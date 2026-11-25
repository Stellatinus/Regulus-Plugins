# hello-world

The test plugin. It does nothing useful, on purpose.

**What it will do:** serve the plugin contract, count its starts in its data directory (`REGULUS_PLUGIN_DATA_DIR`), and be healthy while that directory is writable. Later it *provides* one capability, `hello.greet@1`, so that a second test plugin (`hello-guest`) can *need* it and the store's requirement rules can be tried.

**What it is for:** trying the store end to end before a real plugin depends on it: install, update, rollback, removal, a yanked version.

**What exists today:** the manifest only. **There is no program yet**: a plugin outside the Regulus repository cannot import the SDK (`internal/plugin/sdk`) until it is made importable (see the design, "What blocks the work"). Nothing here has been built or run.

**Version `0.1.0-alpha.1`**: an alpha, offered only to hubs that follow `alpha`.

**It is removed** once the first real plugin is released. Its past versions stay in the index as yanked.
