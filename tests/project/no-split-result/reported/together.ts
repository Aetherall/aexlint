type Link = { label?: string; url?: string };

export const pipelineLink = (path: string | undefined, internal: boolean): Link => {
  if (!path) return {};
  if (internal) return { label: path, url: `/pipelines/${path}` };
  return { label: path, url: `https://ci.example/7314/${path}` };
};
