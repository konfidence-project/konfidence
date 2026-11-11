const test = require('node:test');
const assert = require('node:assert/strict');
const cleanup = require('./cleanup-e2e-packages.cjs');
const now = Date.parse('2026-11-11T12:00:00Z');
const old = '2026-11-09T12:00:00Z';
const recent = '2026-11-11T11:00:00Z';
const pkg = {id: 1, name: 'konfidence/e2e/run-123-1/github.com/example/vector', package_type: 'container', updated_at: old,
  version_count: 1, repository: {full_name: 'konfidence-project/konfidence'}};
const version = {created_at: old, updated_at: old};

async function exercise(options = {}) {
  const candidate = {...pkg, ...options.pkg};
  const calls = [];
  const warnings = [];
  const github = {
    paginate: async (route, params) => {
      calls.push({route, params});
      if (options.fail === route) throw {status: options.status || 403};
      return route.endsWith('/versions') ? (options.versions || [version]) : (options.packages || [candidate]);
    },
    request: async (route, params) => {
      calls.push({route, params});
      if (options.fail === route) throw {status: options.status || 403};
      return {data: route.includes('/actions/runs/') ?
        {id: Number(params.run_id), status: 'completed', path: '.github/workflows/e2e.yaml',
          repository: {full_name: 'konfidence-project/konfidence'}, ...options.run} :
        {...(options.packages?.find(item => item.name === params.package_name) || candidate), ...options.fresh}};
    },
  };
  const result = await cleanup({github, context: {repo: {owner: 'konfidence-project', repo: 'konfidence'}},
    core: {info: () => {}, warning: message => warnings.push(message)}, now,
    dryRun: options.defaultDryRun ? undefined : (options.dryRun ?? false), limit: options.limit});
  return {result, calls, warnings, deletes: calls.filter(call => call.route.startsWith('DELETE '))};
}

const cases = [
  ['completed old run', {}, 1],
  ['recent package', {pkg: {updated_at: recent}}, 0],
  ['exact cutoff is retained', {pkg: {updated_at: '2026-11-10T12:00:00Z'}}, 0],
  ['invalid timestamp', {pkg: {updated_at: 'unknown'}}, 0],
  ['missing timestamp', {pkg: {updated_at: undefined}}, 0],
  ['missing package identity', {pkg: {id: undefined}}, 0],
  ['release package', {pkg: {name: 'konfidence-operator'}}, 0],
  ['source example app', {pkg: {name: 'example-app/candidates'}}, 0],
  ['prefix lookalike', {pkg: {name: 'konfidence/e2e/run-123-1extra/vector'}}, 0],
  ['non-container package', {pkg: {package_type: 'npm'}}, 0],
  ['other repository link', {pkg: {repository: {full_name: 'konfidence-project/example-app'}}}, 0],
  ['missing repository link uses verified run', {pkg: {repository: undefined}}, 1],
  ['active run', {run: {status: 'in_progress'}}, 0],
  ['queued run', {run: {status: 'queued'}}, 0],
  ['waiting run', {run: {status: 'waiting'}}, 0],
  ['unknown run', {fail: 'GET /repos/{owner}/{repo}/actions/runs/{run_id}', status: 404}, 0],
  ['different workflow', {run: {path: '.github/workflows/main.yaml'}}, 0],
  ['unknown workflow', {run: {path: undefined}}, 0],
  ['different run identity', {run: {id: 999}}, 0],
  ['different run repository', {run: {repository: {full_name: 'konfidence-project/example-app'}}}, 0],
  ['new version on a later page', {versions: [version, {created_at: recent, updated_at: recent}]}, 0],
  ['recent version update', {versions: [{created_at: old, updated_at: recent}]}, 0],
  ['unknown version date', {versions: [{created_at: old}]}, 0],
  ['all paged versions are old', {versions: [version, version], pkg: {version_count: 2}}, 1],
  ['publication during sweep', {fresh: {updated_at: recent}}, 0],
  ['version count changed during sweep', {fresh: {version_count: 2}}, 0],
  ['replacement package', {fresh: {id: 2}}, 0],
  ['package moved', {fresh: {name: 'konfidence-operator'}}, 0],
  ['dry run', {dryRun: true}, 0],
  ['dry run is the default', {defaultDryRun: true}, 0],
];
for (const [name, options, expected] of cases) {
  test(name, async () => assert.equal((await exercise(options)).deletes.length, expected));
}

test('pagination, run caching, and bounded batches', async () => {
  const second = {...pkg, id: 2, name: pkg.name.replace('/vector', '/candidates')};
  const full = await exercise({packages: [pkg, second]});
  assert.equal(full.deletes.length, 2);
  assert.equal(full.calls.filter(call => call.route.includes('/actions/runs/')).length, 1);
  assert.ok(full.calls.filter(call => call.route.endsWith('/versions')).every(call => call.params.per_page === 100));
  assert.equal((await exercise({packages: [pkg, second], limit: 1})).deletes.length, 1);
});
test('404 races are tolerated and API failures warn', async () => {
  const route = 'DELETE /orgs/{org}/packages/{package_type}/{package_name}';
  const race = await exercise({fail: route, status: 404});
  assert.equal(race.result.deleted, 0);
  assert.equal(race.warnings.length, 0);
  assert.equal((await exercise({fail: route})).warnings.length, 1);
  assert.equal((await exercise({fail: 'GET /orgs/{org}/packages'})).deletes.length, 0);
});
