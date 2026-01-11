{
  description = "nxretry";

  inputs = {
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = inputs:
    inputs.flake-parts.lib.mkFlake {inherit inputs;} {
      systems = inputs.nixpkgs.lib.systems.flakeExposed;

      imports = [inputs.treefmt-nix.flakeModule];

      # Per-system attributes.
      #
      # This generates `name`.${system} attrsets in a convinent way.
      perSystem = {
        pkgs,
        system,
        ...
      }: {
        _module.args.pkgs = import inputs.nixpkgs {inherit system;};

        # Configure the default devShell with common development dependencies.
        devShells.default = pkgs.mkShellNoCC {
          packages = with pkgs; [
            go_1_25
            gofumpt
            gotools
          ];
        };

        # treefmt configuration, used to format the entire repository tree.
        #
        # treefmt is called when `nix fmt` is ran.
        treefmt = {
          projectRootFile = "flake.nix";

          settings.global.excludes = [
            ".editorconfig"
            "LICENSE"
          ];

          programs = {
            # Enable alejandra, a Nix formatter.
            alejandra.enable = true;

            # Enable deadnix, a Nix linter/formatter that removes unused Nix code.
            deadnix.enable = true;

            # Enable gofumpt, a Golang formatter.
            gofumpt = {
              enable = true;
              extra = true;
            };

            # Enable prettier, a multi-language formatter primarily used for JavaScript.
            prettier = {
              enable = true;
              includes = ["*.md"];
            };

            # Enable yamlfmt, a YAML formatter.
            yamlfmt = {
              enable = true;
              settings.formatter = {
                type = "basic";
                retain_line_breaks_single = true;
              };
            };
          };
        };
      };
    };
}
