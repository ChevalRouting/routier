export const SITE_URL = process.env.SITE_URL ?? 'https://routier.cloud';
export const REPO_URL = process.env.REPO_URL ?? 'https://repo.routier.cloud';
export const GITHUB_URL = process.env.GITHUB_URL ?? 'https://github.com/ChevalRouting/routier';

const tokens: Record<string, string> = {
  '%%SITE_URL%%': SITE_URL,
  '%%REPO_URL%%': REPO_URL,
  '%%GITHUB_URL%%': GITHUB_URL,
};

interface MdastNode {
  value?: string;
  children?: MdastNode[];
}

function substitute(node: MdastNode): void {
  if (typeof node.value === 'string') {
    for (const [token, value] of Object.entries(tokens)) {
      if (node.value.includes(token)) {
        node.value = node.value.split(token).join(value);
      }
    }
  }
  node.children?.forEach(substitute);
}

export default function remarkDomains() {
  return (tree: MdastNode): void => {
    substitute(tree);
  };
}
