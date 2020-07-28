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

/* A webpack config file which builds the chrome extension */

const CopyPlugin = require('copy-webpack-plugin');
const ExtensionReloader = require('webpack-extension-reloader');
const srcDir = './chrome/src/';
const path = require('path');
const webpack = require('webpack');

module.exports = { 
  // Entry files are the TypeScript files that are the extension.
  entry: {
    background: path.join(__dirname, srcDir + 'background.ts'),
  },
  output: {
    path: path.join(__dirname, 'dist'),
    filename: '[name].js'
  },
  devtool: 'inline-source-map',
  optimization: {
    splitChunks: {
      name: 'vendor',
      chunks: 'initial',
    },
  },
  resolve: {
    // Add `.ts` as a resolvable extension.
    extensions: ['.ts', '.js'],
  },
  module: {
    rules: [
      // all files with a `.ts` or `.tsx` extension will be handled by `ts-loader`
      {test: /\.tsx?$/, loader: 'ts-loader', exclude: '/node_modules/'},
    ],
  },
  plugins: [
    // Copies the base manifest JSON to the /dist folder.
    // Adds the extension's key to the manifest from ENV variables.
    new CopyPlugin({
      patterns: [
        {
          from: 'chrome/manifest.json',
          to: './',
          transform: function (content, path) {
            // generates the manifest file using the package.json informations
            return Buffer.from(
              JSON.stringify({
                key: process.env.EXTENSION_DEV_KEY,
                ...JSON.parse(content.toString()),
              })
            );
          },
        },
      ],
    }),

    // Initializes the automatic reloading of the chrome extension during development.
    new ExtensionReloader({
      entries: {
        background: 'background',
      },
    }),
  ]
};
