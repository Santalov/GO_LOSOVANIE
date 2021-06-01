import createMuiTheme from "@material-ui/core/styles/createMuiTheme";
import appTypography from "./app-typography";

export default createMuiTheme(
  {
    palette: {
      type: "dark",
      primary: {
        light: "#5ab7ff",
        main: "#0097ff",
        dark: "#0e75eb",
        contrastText: "#ffffff",
      },
      secondary: {
        light: "#6dd2c4",
        main: "#00b09a",
        dark: "#00937a",
        contrastText: "#ffffff",
      },
      error: {
        light: "#ff8a65",
        main: "#ff5722",
        dark: "#e64a19",
        contrastText: "#ffffff",
      },
      success: {
        light: "#6dd2c4",
        main: "#00b09a",
        dark: "#00937a",
        contrastText: "#ffffff",
      },
    },
  },
  appTypography
);
