import React, {PropsWithChildren} from "react";
import {createStyles, makeStyles, Theme} from "@material-ui/core";

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    container: {
      padding: theme.spacing(3),
      display: "flex",
      alignItems: "center",
      minHeight: "100%",
    },
    content: {
      width: "100%",
    },
  })
);

function AppContainer({children}: PropsWithChildren<{}>) {
  const classes = useStyles();
  return (
    <div className={classes.container}>
      <div className={classes.content}>{children}</div>
    </div>
  );
}

export default AppContainer;
