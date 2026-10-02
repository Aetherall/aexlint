{ pkgs, ... }:
{
  languages.javascript = {
    enable = true;
    package = pkgs.nodejs_24;
    pnpm.enable = true;
  };

  languages.go = {
    enable = true;
    package = pkgs.go_1_26;
  };

  packages = [ pkgs.patch pkgs.gnutar ];

  enterTest = ''
    pnpm install --frozen-lockfile
    pnpm check
  '';
}
