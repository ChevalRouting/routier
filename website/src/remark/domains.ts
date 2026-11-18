export const SITE_URL = process.env.SITE_URL ?? 'https://routier.chevalrouting.org';
export const REPO_URL = process.env.REPO_URL ?? 'https://routier.chevalrouting.org/repo';
export const GITHUB_URL = process.env.GITHUB_URL ?? 'https://github.com/ChevalRouting/routier';

const tokens: Record<string, string> = {
  '%%SITE_URL%%': SITE_URL,
  '%%REPO_URL%%': REPO_URL,
  '%%GITHUB_URL%%': GITHUB_URL,
};

interface MdastNode {
  value?: string;
  url?: string;
  children?: MdastNode[];
}

function replaceTokens(text: string): string {
  return Object.entries(tokens).reduce((out, [token, value]) => out.split(token).join(value), text);
}

function substitute(node: MdastNode): void {
  if (typeof node.value === 'string') {
    node.value = replaceTokens(node.value);
  }
  if (typeof node.url === 'string') {
    node.url = replaceTokens(node.url);
  }
  node.children?.forEach(substitute);
}

export default function remarkDomains() {
  return (tree: MdastNode): void => {
    substitute(tree);
  };
}
