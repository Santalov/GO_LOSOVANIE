import React from "react";
import withStyles from "@material-ui/core/styles/withStyles";
import AppLogo from "../app-logo/app-logo";

const styles = (theme) => {
  const size = 40;
  const largeSize = size * 2;
  return {
    container: {
      display: "grid",
      gridTemplateColumns: "auto 1fr",
      gridColumnGap: theme.spacing(1),
      alignItems: "center",
      height: size,
      fontWeight: 300,
    },
    logo: {
      width: size,
      height: size,
    },
    name: {
      fontSize: size / 2 + "px",
      lineHeight: size / 2 + "px",
      fontWeight: "400",
    },
    largeContainer: {
      height: largeSize,
    },
    largeLogo: {
      height: largeSize,
      width: largeSize,
    },
    largeName: {
      fontSize: largeSize / 2 + "px",
      lineHeight: largeSize / 2 + "px",
    },
  };
};

function AppLogoGroupRaw({classes, large}: { classes?: any, large?: boolean }) {
  const largeStr = large || "";
  return (
    <div
      className={classes.container + (largeStr && " " + classes.largeContainer)}
    >
      <div className={classes.logo + (largeStr && " " + classes.largeLogo)}>
        <AppLogo/>
      </div>
      <div className={classes.name + (largeStr && " " + classes.largeName)}>
        <div>GO</div>
        <div>LOSOVANIE</div>
      </div>
    </div>
  );
}

// @ts-ignore
const AppLogoGroup = withStyles(styles)(AppLogoGroupRaw);

export default AppLogoGroup;
