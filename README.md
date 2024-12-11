# PixieDust ✨ - High throughput image processing application
PixieDust is a quick and dirty solution to process common image formats into WebP (with support for animated webp too). I designed this project mainly as a POC for Nerimity as their current solution can cause CDN crashes, however this is pretty simple to use. (hi superkitten wsg pooks)

You'll need to edit the code to edit how PixieDust works, however here's how it's configured by default:
- Max size: 1920x1080
- 50% image quality
- Outputs .webp files

Due to the project's use of `discord/lilliput` for the actual image processing, this project only supports OSX/Linux.

## Usage:
> ./pixiedust \<image path\> \<destination path\>