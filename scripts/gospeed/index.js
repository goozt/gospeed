#!/usr/bin/env node
const { spawnSync } = require('child_process');

// Dynamically find the path to the binary package
const pkgName = `@goozt/gospeed-${process.platform}-${process.arch}`;
try {
    const binPath = require.resolve(`${pkgName}/gospeed`);
    const result = spawnSync(binPath, process.argv.slice(2), { stdio: 'inherit' });
    process.exit(result.status);
} catch (e) {
    console.error(`Error: Could not find binary for ${process.platform}-${process.arch}`);
    process.exit(1);
}