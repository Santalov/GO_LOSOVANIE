import React, {PropsWithChildren} from "react";
import makeStyles from "@material-ui/core/styles/makeStyles";
import {createStyles, Theme} from "@material-ui/core";

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    root: {
      backgroundColor: theme.palette.background.default,
      color: theme.palette.text.primary,
      height: "100%",
      overflow: "auto",
    },
  })
);

function AppBackground({children}: PropsWithChildren<{}>) {
  const classes = useStyles();
  return <div className={classes.root}>{children}</div>;
}

export default AppBackground;
