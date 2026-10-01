{
  description = "kcore-migrate development shell";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      each = f: nixpkgs.lib.genAttrs systems (system: f (import nixpkgs { inherit system; }));
    in
    {
      devShells = each (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            git
            gnumake
            gh
            cacert
            coreutils
          ] ++ lib.optionals stdenv.hostPlatform.isLinux [
            # virt-v2v links the nixpkgs virtio-win tree for Windows guests.
            virt-v2v
            qemu-utils
          ];
        };
      });
    };
}
