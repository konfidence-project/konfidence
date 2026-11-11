module.exports = async ({github, context, core, dryRun = true, now = Date.now(), limit = 200}) => {
  const {owner, repo} = context.repo;
  const prefix = `${repo.toLowerCase()}/e2e/run-`;
  const fullName = `${owner}/${repo}`.toLowerCase();
  const cutoff = now - 24 * 60 * 60 * 1000;
  const stale = value => typeof value === 'string' && Date.parse(value) < cutoff;
  const scoped = pkg => Number.isSafeInteger(pkg?.id) && pkg.id > 0 && pkg.package_type === 'container' && typeof pkg.name === 'string' &&
    pkg.name.startsWith(prefix) && /^([1-9]\d*)-([1-9]\d*)\/.+$/.test(pkg.name.slice(prefix.length)) &&
    (!pkg.repository || (typeof pkg.repository.full_name === 'string' && pkg.repository.full_name.toLowerCase() === fullName));
  let packages;
  try {
    packages = await github.paginate('GET /orgs/{org}/packages', {
      org: owner, package_type: 'container', per_page: 100,
    });
  } catch (error) {
    core.warning(`Cannot list e2e packages (HTTP ${error.status || 'unknown'}).`);
    return;
  }
  if (!Array.isArray(packages)) {
    core.warning('Invalid e2e package inventory.');
    return;
  }
  const candidates = packages.filter(pkg => scoped(pkg) && stale(pkg.updated_at))
    .sort((a, b) => Date.parse(a.updated_at) - Date.parse(b.updated_at)).slice(0, limit);
  const runs = new Map();
  let selected = 0;
  let deleted = 0;
  for (const pkg of candidates) {
    try {
      const runId = pkg.name.slice(prefix.length).split('-')[0];
      if (!runs.has(runId)) {
        const {data} = await github.request('GET /repos/{owner}/{repo}/actions/runs/{run_id}', {
          owner, repo, run_id: runId,
        });
        runs.set(runId, data);
      }
      const run = runs.get(runId);
      if (String(run.id) !== runId || run.status !== 'completed' ||
        run.repository?.full_name?.toLowerCase() !== fullName || run.path?.split('@')[0] !== '.github/workflows/e2e.yaml') continue;
      const params = {org: owner, package_type: 'container', package_name: pkg.name};
      const versions = await github.paginate('GET /orgs/{org}/packages/{package_type}/{package_name}/versions', {
        ...params, per_page: 100,
      });
      if (!versions.every(version => stale(version.created_at) && stale(version.updated_at))) continue;
      const {data: fresh} = await github.request('GET /orgs/{org}/packages/{package_type}/{package_name}', params);
      if (fresh.id !== pkg.id || fresh.name !== pkg.name || fresh.version_count !== versions.length ||
        !scoped(fresh) || !stale(fresh.updated_at)) continue;
      selected++;
      if (dryRun) {
        core.info(`Would delete ${JSON.stringify(pkg.name)}`);
      } else {
        await github.request('DELETE /orgs/{org}/packages/{package_type}/{package_name}', params);
        deleted++;
        core.info(`Deleted ${JSON.stringify(pkg.name)}`);
      }
    } catch (error) {
      if (error.status !== 404) core.warning(`Skipped ${JSON.stringify(pkg.name)} (HTTP ${error.status || 'unknown'}).`);
    }
  }
  core.info(`E2E package cleanup: ${candidates.length}/${limit} aged candidates checked, ${selected} selected, ${deleted} deleted.`);
  return {checked: candidates.length, selected, deleted};
};
