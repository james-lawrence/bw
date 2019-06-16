### rust setup
- yay -Syu rustup
- rustup default stable
- rustup target add wasm32-unknown-unknown
- rustc --target wasm32-unknown-unknown .bw/.remote/01_wasm.rs
