module.exports = {
  root: true,
  ignorePatterns: ["dist", "node_modules", "miniapp/frontend/dist"],
  overrides: [
    {
      files: ["miniapp/frontend/**/*.{ts,tsx}"],
      extends: ["taro/react"],
      parserOptions: {
        tsconfigRootDir: __dirname,
        project: ["./miniapp/frontend/tsconfig.json"]
      },
      rules: {
        "react/react-in-jsx-scope": "off",
        "react/jsx-uses-react": "off"
      }
    }
  ]
};
