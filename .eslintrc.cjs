module.exports = {
  root: true,
  ignorePatterns: ["dist", "node_modules", "frontend/miniapp/dist"],
  overrides: [
    {
      files: ["frontend/miniapp/**/*.{ts,tsx}"],
      extends: ["taro/react"],
      parserOptions: {
        tsconfigRootDir: __dirname,
        project: ["./frontend/miniapp/tsconfig.json"]
      },
      rules: {
        "react/react-in-jsx-scope": "off",
        "react/jsx-uses-react": "off"
      }
    }
  ]
};
