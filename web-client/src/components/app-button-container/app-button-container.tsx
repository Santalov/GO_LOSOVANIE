import React, {PropsWithChildren} from "react";
import {createStyles, makeStyles, Theme} from '@material-ui/core';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    container: {
      display: "flex",
      justifyContent: "flex-end",
      alignItems: "stretch",
    },
  })
);

function AppButtonContainer({children, ...props}: PropsWithChildren<{}>) {
  const classes = useStyles();
  return (
    <div className={classes.container} {...props}>
      {children}
    </div>
  );
}

export default AppButtonContainer;
