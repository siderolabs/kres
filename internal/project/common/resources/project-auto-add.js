// Adds the pull request to the organization project, sets its status based on the draft state,
// and assigns it to the current iteration.
//
// All inputs come from the environment and the event payload, nothing is interpolated into the script.

const pr = context.payload.pull_request;
const action = context.payload.action;

const statusName = pr.draft ? process.env.PROJECT_DRAFT_STATUS : process.env.PROJECT_READY_STATUS;

const { organization } = await github.graphql(
  `query($owner: String!, $number: Int!, $statusField: String!, $iterationField: String!) {
    organization(login: $owner) {
      projectV2(number: $number) {
        id
        status: field(name: $statusField) {
          ... on ProjectV2SingleSelectField {
            id
            options { id name }
          }
        }
        iteration: field(name: $iterationField) {
          ... on ProjectV2IterationField {
            id
            configuration {
              iterations { id title startDate duration }
            }
          }
        }
      }
    }
  }`,
  {
    owner: process.env.PROJECT_OWNER,
    number: parseInt(process.env.PROJECT_NUMBER, 10),
    statusField: process.env.PROJECT_STATUS_FIELD,
    iterationField: process.env.PROJECT_ITERATION_FIELD,
  },
);

const project = organization?.projectV2;

if (!project) {
  throw new Error(`project ${process.env.PROJECT_OWNER}/${process.env.PROJECT_NUMBER} not found`);
}

if (!project.status?.id) {
  throw new Error(`single select field "${process.env.PROJECT_STATUS_FIELD}" not found in the project`);
}

if (!project.iteration?.id) {
  throw new Error(`iteration field "${process.env.PROJECT_ITERATION_FIELD}" not found in the project`);
}

const statusOption = project.status.options.find((option) => option.name === statusName);

if (!statusOption) {
  throw new Error(`status "${statusName}" not found in the field "${process.env.PROJECT_STATUS_FIELD}"`);
}

// adding an item which is already in the project returns the existing item
const { addProjectV2ItemById: { item } } = await github.graphql(
  `mutation($project: ID!, $content: ID!, $iterationField: String!) {
    addProjectV2ItemById(input: {projectId: $project, contentId: $content}) {
      item {
        id
        iteration: fieldValueByName(name: $iterationField) {
          ... on ProjectV2ItemFieldIterationValue { iterationId }
        }
      }
    }
  }`,
  {
    project: project.id,
    content: pr.node_id,
    iterationField: process.env.PROJECT_ITERATION_FIELD,
  },
);

core.info(`pull request #${pr.number} is project item ${item.id}`);

const setField = async (fieldId, value) => {
  await github.graphql(
    `mutation($project: ID!, $item: ID!, $field: ID!, $value: ProjectV2FieldValue!) {
      updateProjectV2ItemFieldValue(input: {projectId: $project, itemId: $item, fieldId: $field, value: $value}) {
        projectV2Item { id }
      }
    }`,
    {
      project: project.id,
      item: item.id,
      field: fieldId,
      value,
    },
  );
};

await setField(project.status.id, { singleSelectOptionId: statusOption.id });

core.info(`status set to "${statusName}"`);

// (re)opened pull requests go to the current iteration, while draft state changes keep
// the iteration which might have been changed manually (unless it is not set at all)
if (action === 'opened' || action === 'reopened' || !item.iteration?.iterationId) {
  const dayMs = 24 * 60 * 60 * 1000;
  const now = Date.now();

  const current = project.iteration.configuration.iterations.find((iteration) => {
    const start = Date.parse(`${iteration.startDate}T00:00:00Z`);

    return start <= now && now < start + iteration.duration * dayMs;
  });

  if (current) {
    await setField(project.iteration.id, { iterationId: current.id });

    core.info(`iteration set to "${current.title}"`);
  } else {
    core.warning('no current iteration found in the project, iteration is not set');
  }
}
