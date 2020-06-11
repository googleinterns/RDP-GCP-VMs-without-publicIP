/***
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
***/

const CopyPlugin = require("copy-webpack-plugin");
const ExtensionReloader = require("webpack-extension-reloader");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const path = require("path");
const srcDir = "./src/";
const webpack = require("webpack");

module.exports = {
    entry: {
        popup: path.join(__dirname, srcDir + "popup.ts"),
        background: path.join(__dirname, srcDir + "background.ts"),
        content_script: path.join(__dirname, srcDir + "content.ts")
    },
    output: {
        path: path.join(__dirname, './dist/js'),
        filename: "[name].js"
    },
    devtool: 'cheap-module-source-map',
    optimization: {
        splitChunks: {
            name: "vendor",
            chunks: "initial"
        }
    },
    resolve: {
        // Add `.ts` and `.tsx` as a resolvable extension.
        extensions: [".ts", ".tsx", ".js"]
    },
    module: {
        rules: [
            // all files with a `.ts` or `.tsx` extension will be handled by `ts-loader`
            { test: /\.tsx?$/, loader: "ts-loader", exclude: "/node_modules/" }
        ]
    },
    plugins: [
        new CopyPlugin({
            patterns: [{
                from: "public/manifest.json",
                to: "../",
                transform: function (content, path) {
                  // generates the manifest file using the package.json informations
                  return Buffer.from(JSON.stringify({
                    key: process.env.EXTENSION_DEV_KEY,
                    ...JSON.parse(content.toString())
                  }))
                }
            }]
        }),
        new ExtensionReloader({
            entries: {
                background: 'background'
            }
        }),
        new HtmlWebpackPlugin({
            template: path.join(__dirname, "public", "popup.html"),
            filename: "../popup.html",
            chunks: ["popup"]
        }),
        new webpack.BannerPlugin({
            banner: `
            Copyright 2020 Google LLC

            Licensed under the Apache License, Version 2.0 (the "License");
            you may not use this file except in compliance with the License.
            You may obtain a copy of the License at

                https://www.apache.org/licenses/LICENSE-2.0

            Unless required by applicable law or agreed to in writing, software
            distributed under the License is distributed on an "AS IS" BASIS,
            WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
            See the License for the specific language governing permissions and
            limitations under the License.
            `
        })
    ]
}
