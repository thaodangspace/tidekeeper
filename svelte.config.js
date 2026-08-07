import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  kit: {
    adapter: adapter({ strict: true }),
    prerender: { entries: ['*'] }
  }
};

export default config;
