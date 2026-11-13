import { writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { defineConfig } from 'vite';

// Go embeds frontend/dist, so the folder must exist even before the first
// build. Vite empties the output folder on every build, which would delete
// the placeholder. This plugin puts it back.
function keepGitkeep() {
  return {
    name: 'keep-gitkeep',
    closeBundle() {
      writeFileSync(resolve(__dirname, 'dist', '.gitkeep'), '');
    },
  };
}

export default defineConfig({
  plugins: [keepGitkeep()],
  build: {
    // The Wails asset server serves these from the embedded FS.
    outDir: 'dist',
    emptyOutDir: true,
    target: 'esnext',
  },
  server: {
    port: 34115,
    strictPort: true,
  },
});
