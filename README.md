# [Dantotsu]("https://github.com/rebelonion/Dantotsu/tree/dev") Updater

This is a simple script that grabs the latest release from the actions artifact of the last run, downloads the zip file, grabs the pretester apk from it, the uploads it to releases.

This is written in [go](https://go.dev/) and utilizes github workflow to run the script continuously.

## Usage

Download [Obtainium](https://github.com/ImranR98/Obtainium) (click the name its a link)

then add this repository link to Obtainium.\
Then for the settings:

1. Set latest asset upload as release date to `ON`.
2. Set release date as version string (pseudo-version) to `ON`.
3. Set Reconcile version string with version detected OS to `OFF`.

## Credits

-   [ibo](https://github.com/sneazy-ibo) - fixed\fixing some workflow issues.
-   [me](https://github.com/kaorlol) - all of the go code
-   [chatgpt](https://chat.openai.com/) - most of the workflow shit because its aids 😡😡😡

