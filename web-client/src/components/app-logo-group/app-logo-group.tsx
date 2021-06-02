import React from "react";
import AppLogo from "../app-logo/app-logo";
import {createStyles, makeStyles, Theme} from '@material-ui/core';

const useStyles = makeStyles((theme: Theme) => {
    const size = 40;
    const largeSize = size * 2;
    return createStyles({
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
        fontWeight: 'normal',
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
    })
  }
);

function AppLogoGroup({large}: { large?: boolean }) {
  const largeStr = large || "";
  const classes = useStyles();
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

export default AppLogoGroup;
