module.exports = {
  root: true,
  ignorePatterns: ["dist", "node_modules", "src/frontend/miniapp/dist"],
  overrides: [
    {
      files: ["src/frontend/miniapp/**/*.{ts,tsx}"],
      extends: ["taro/react"],
      parserOptions: {
        tsconfigRootDir: __dirname,
        project: ["./src/frontend/miniapp/tsconfig.json"]
      },
      rules: {
        "react/react-in-jsx-scope": "off",
        "react/jsx-uses-react": "off"
      }
    }
  ]
};
