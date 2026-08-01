/**
 * Upgrade-tree validation and normalization (ported from keeper/upgrade_tree.go).
 */

export interface UpgradeNode {
  key: string;
  depth: number;
  isRoot: boolean;
}

const contentKeyPattern = /^[a-z][a-z0-9_]*$/;

function isJsonObject(text: string): boolean {
  try {
    const value = JSON.parse(text);
    return value !== null && typeof value === "object" && !Array.isArray(value);
  } catch {
    return false;
  }
}

interface ParsedTree {
  rootNodeKey: string;
  nodes: { key: string; name: string; next: string[]; effects: string }[];
}

/**
 * Validates an upgrade tree and returns normalized node projections with
 * zero-based depth from the root, in deterministic key order.
 */
export function parseUpgradeTree(tree: string): UpgradeNode[] {
  let parsed: ParsedTree;
  try {
    parsed = JSON.parse(tree);
  } catch {
    throw new Error("upgrade tree must be a JSON object");
  }
  if (
    !isJsonObject(tree) || typeof parsed.rootNodeKey !== "string" ||
    !Array.isArray(parsed.nodes) || parsed.nodes.length === 0
  ) {
    throw new Error("upgrade tree must be a JSON object");
  }
  if (!contentKeyPattern.test(parsed.rootNodeKey)) {
    throw new Error("upgrade tree requires a valid rootNodeKey and nodes");
  }

  const nextByKey = new Map<string, string[]>();
  const inDegree = new Map<string, number>();
  for (const node of parsed.nodes) {
    if (!contentKeyPattern.test(node.key)) {
      throw new Error(`upgrade node ${node.key} has an invalid key`);
    }
    if (node.name === "" || !isJsonObject(node.effects)) {
      throw new Error(
        `upgrade node ${node.key} must have a name and object effects`,
      );
    }
    if (nextByKey.has(node.key)) {
      throw new Error(`duplicate upgrade node key ${node.key}`);
    }
    nextByKey.set(node.key, node.next);
    inDegree.set(node.key, 0);
  }
  if (!nextByKey.has(parsed.rootNodeKey)) {
    throw new Error(`upgrade root ${parsed.rootNodeKey} is not a node`);
  }
  for (const [nodeKey, next] of nextByKey) {
    for (const childKey of next) {
      if (!nextByKey.has(childKey)) {
        throw new Error(
          `upgrade node ${nodeKey} references unknown node ${childKey}`,
        );
      }
      inDegree.set(childKey, (inDegree.get(childKey) ?? 0) + 1);
    }
  }
  if (inDegree.get(parsed.rootNodeKey) !== 0) {
    throw new Error("upgrade root has an incoming edge");
  }
  for (const [nodeKey, degree] of inDegree) {
    if (nodeKey !== parsed.rootNodeKey && degree !== 1) {
      throw new Error(`upgrade node ${nodeKey} must have exactly one parent`);
    }
  }

  const visiting = new Set<string>();
  const visited = new Set<string>();
  const visit = (nodeKey: string): void => {
    if (visiting.has(nodeKey)) {
      throw new Error(`upgrade tree contains cycle at ${nodeKey}`);
    }
    if (visited.has(nodeKey)) {
      return;
    }
    visiting.add(nodeKey);
    for (const childKey of nextByKey.get(nodeKey) ?? []) {
      visit(childKey);
    }
    visiting.delete(nodeKey);
    visited.add(nodeKey);
  };
  visit(parsed.rootNodeKey);
  if (visited.size !== nextByKey.size) {
    const missing = [...nextByKey.keys()].filter((key) => !visited.has(key))
      .sort()[0]!;
    throw new Error(`upgrade tree is disconnected at ${missing}`);
  }

  const depthByKey = new Map<string, number>([[parsed.rootNodeKey, 0]]);
  const queue = [parsed.rootNodeKey];
  for (let head = 0; head < queue.length; head++) {
    for (const childKey of nextByKey.get(queue[head]!) ?? []) {
      if (depthByKey.has(childKey)) {
        continue;
      }
      depthByKey.set(childKey, depthByKey.get(queue[head]!)! + 1);
      queue.push(childKey);
    }
  }

  const nodes: UpgradeNode[] = [];
  for (const key of nextByKey.keys()) {
    nodes.push({
      key,
      depth: depthByKey.get(key)!,
      isRoot: key === parsed.rootNodeKey,
    });
  }
  nodes.sort((a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0));
  return nodes;
}
