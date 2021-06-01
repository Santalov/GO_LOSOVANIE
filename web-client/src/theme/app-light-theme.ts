import createMuiTheme from "@material-ui/core/styles/createMuiTheme";
import appTypography from "./app-typography";

export default createMuiTheme(
  {
    palette: {
      type: "light",
      primary: {
        light: "#4f1aff",
        main: "#0008ff",
        dark: "#0006dd",
        contrastText: "#ffffff",
      },
      secondary: {
        light: "#00d2c8",
        main: "#00b6a6",
        dark: "#009685",
        contrastText: "#ffffff",
      },
      error: {
        light: "#e57373",
        main: "#f44336",
        dark: "#d32f2f",
        contrastText: "#ffffff",
      },
      success: {
        light: "#00d2c8",
        main: "#00b6a6",
        dark: "#009685",
        contrastText: "#ffffff",
      },
      background: {
        default: "#ffffff",
        paper: "#ffffff",
      },
    },
  },
  appTypography
);
