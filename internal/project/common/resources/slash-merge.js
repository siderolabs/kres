// Validates a `/nm` comment and, if the PR can be merged, outputs the PR head SHA
// the base branch should be fast-forwarded to.
//
// Runs from the default branch (issue_comment event), never executes any PR code.
// Nothing from the event or the API is ever interpolated into code: all untrusted
// values (comment body, check names, branch names) are handled as data.
const { owner, repo } = context.repo;
const issueNumber = context.payload.issue.number;
const actor = context.payload.comment.user.login;

// GitHub's legacy permission field maps 'maintain' to 'write' and 'triage' to 'read'.
const writePermissions = ['admin', 'maintain', 'write'];

const hasWriteAccess = async (username) => {
  try {
    const { data } = await github.rest.repos.getCollaboratorPermissionLevel({ owner, repo, username });

    return writePermissions.includes(data.permission);
  } catch (e) {
    core.warning(`failed to get permission level for ${username}: ${e.message}`);

    return false;
  }
};

const react = (content) => github.rest.reactions.createForIssueComment({
  owner,
  repo,
  comment_id: context.payload.comment.id,
  content,
});

// check run and status names come from the PR's workflows, so they are attacker-controlled for PRs from forks
const quote = (s) => '`' + String(s).replace(/[`\r\n]/g, ' ').slice(0, 100) + '`';

const reject = async (message) => {
  await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
    body: `⛔ @${actor} ${message}`,
  });

  core.setFailed(message);
};

// The job-level condition is only a cheap pre-filter, require an exact command on the first line.
const command = context.payload.comment.body.split(/\r?\n/, 1)[0].trim();

if (command !== '/nm') {
  core.info(`not a merge command: ${JSON.stringify(command)}`);

  return;
}

if (!(await hasWriteAccess(actor))) {
  // no comment for unauthorized users, so that the workflow can't be used to spam the PR
  await react('-1');
  core.setFailed(`${actor} doesn't have write access to the repository`);

  return;
}

await react('eyes');

// mergeable_state is computed asynchronously, poll until it is known
let pr;

for (let attempt = 0; attempt < 10; attempt++) {
  ({ data: pr } = await github.rest.pulls.get({ owner, repo, pull_number: issueNumber }));

  if (pr.mergeable_state !== 'unknown') {
    break;
  }

  await new Promise((r) => setTimeout(r, 3000));
}

const sha = pr.head.sha;
const base = pr.base.ref;

core.info(`PR #${issueNumber}: head ${sha}, base ${base}, mergeable_state ${pr.mergeable_state}`);

if (pr.state !== 'open' || pr.draft) {
  return reject('the PR is closed or a draft, cannot merge.');
}

switch (pr.mergeable_state) {
  case 'dirty':
    return reject('the PR has merge conflicts, please rebase it.');
  case 'behind':
    return reject(`the PR is behind ${quote(base)}, please rebase it.`);
  case 'blocked':
    return reject('the PR is blocked by branch protection (missing approvals or required checks).');
  case 'unknown':
    return reject('GitHub has not computed the mergeable state of the PR yet, please try again later.');
}

// For PRs from forks the head branch can be updated by users without write access, and the CI
// workflows for such PRs come from the PR itself, so green checks don't prove anything.
// Require an approval of the exact head commit by a user with write access, so that
// commits pushed after the review can't be merged without another review.
const isFork = pr.head.repo === null || pr.head.repo.full_name !== pr.base.repo.full_name;

if (isFork) {
  const reviews = await github.paginate(github.rest.pulls.listReviews, {
    owner,
    repo,
    pull_number: issueNumber,
    per_page: 100,
  });

  // only the latest review of each reviewer counts (reviews are returned in chronological order)
  const latest = new Map();

  for (const review of reviews) {
    if (review.user && ['APPROVED', 'CHANGES_REQUESTED', 'DISMISSED'].includes(review.state)) {
      latest.set(review.user.login, review);
    }
  }

  let approved = false;

  for (const [login, review] of latest) {
    if (review.state === 'APPROVED' && review.commit_id === sha && login !== pr.user.login && (await hasWriteAccess(login))) {
      approved = true;

      break;
    }
  }

  if (!approved) {
    return reject(`the PR is from a fork, it needs an approval of the head commit ${sha.slice(0, 12)} by a maintainer.`);
  }
}

// Check runs and commit statuses of the head commit: everything must be finished and successful.
const checkRuns = await github.paginate(github.rest.checks.listForRef, {
  owner,
  repo,
  ref: sha,
  filter: 'latest',
  per_page: 100,
});

const checkSuites = await github.paginate(github.rest.checks.listSuitesForRef, {
  owner,
  repo,
  ref: sha,
  per_page: 100,
});

const { data: combined } = await github.rest.repos.getCombinedStatusForRef({ owner, repo, ref: sha, per_page: 100 });

const okConclusions = ['success', 'neutral', 'skipped'];

const pending = [
  ...checkRuns.filter((r) => r.status !== 'completed').map((r) => r.name),
  ...combined.statuses.filter((s) => s.state === 'pending').map((s) => s.context),
];

const failed = [
  ...checkRuns.filter((r) => r.status === 'completed' && !okConclusions.includes(r.conclusion)).map((r) => r.name),
  ...combined.statuses.filter((s) => ['failure', 'error'].includes(s.state)).map((s) => s.context),
];

// workflow runs waiting for approval (e.g. first-time contributors) have no check runs yet
const awaitingApproval = checkSuites.filter((s) => s.conclusion === 'action_required').map((s) => s.app ? s.app.slug : 'unknown');

core.info(`check runs: ${checkRuns.map((r) => `${r.name}=${r.conclusion || r.status}`).join(', ') || 'none'}`);
core.info(`statuses: ${combined.statuses.map((s) => `${s.context}=${s.state}`).join(', ') || 'none'}`);

if (awaitingApproval.length > 0) {
  return reject('some workflows are waiting for approval to run, cannot merge.');
}

if (pending.length > 0) {
  return reject(`some checks are still pending: ${pending.map(quote).join(', ')}.`);
}

if (failed.length > 0) {
  return reject(`some checks failed: ${failed.map(quote).join(', ')}.`);
}

const succeeded = new Set([
  ...checkRuns.filter((r) => r.conclusion === 'success').map((r) => r.name),
  ...combined.statuses.filter((s) => s.state === 'success').map((s) => s.context),
]);

// Fail closed: every check required by branch protection must be present and successful.
const { data: branch } = await github.rest.repos.getBranch({ owner, repo, branch: base });
const requiredChecks = branch.protection?.required_status_checks?.contexts ?? [];
const missing = requiredChecks.filter((c) => !succeeded.has(c));

if (missing.length > 0) {
  return reject(`required checks haven't passed: ${missing.map(quote).join(', ')}.`);
}

if (succeeded.size === 0) {
  return reject('no successful checks found for the head commit, cannot merge.');
}

core.setOutput('sha', sha);
core.setOutput('base', base);
