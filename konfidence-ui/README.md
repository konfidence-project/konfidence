# sv

Everything you need to build a Svelte project, powered by [`sv`](https://github.com/sveltejs/cli).

## Creating a project

If you're seeing this, you've probably already done this step. Congrats!

```sh
# create a new project
npx sv create my-app
```

To recreate this project with the same configuration:

```sh
# recreate this project
pnpm dlx sv@0.16.1 create --template minimal --types ts --add vitest="usages:unit,component" playwright sveltekit-adapter="adapter:node" --install pnpm my-app
```

## Developing

Install dependencies from the repository root:

```sh
pnpm install
```

Start a development server:

```sh
make dev-ui

# or start the server and open the app in a new browser tab
pnpm --filter konfidence-ui dev -- --open
```

## Building

To create a production version of your app:

```sh
make build-ui
```

You can preview the production build with `pnpm --filter konfidence-ui preview`.

> To deploy your app, you may need to install an [adapter](https://svelte.dev/docs/kit/adapters) for your target environment.
