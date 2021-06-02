import React, {PropsWithChildren} from "react";
import {createStyles, makeStyles, Theme} from '@material-ui/core';
import classNames from 'classnames';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    errorCard: {
      padding: theme.spacing(1),
      // backgroundColor: theme.palette.error.main,
      border: "1px solid " + theme.palette.error.dark,
      display: "flex",
      alignItems: "center",
      alignContent: "center",
      fontSize: "0.9rem",
      borderRadius: 4,
    },
  })
);

function AppErrorCard({children, className}: PropsWithChildren<{ className?: string }>) {
  const classes = useStyles();
  return (
    <div className={classNames(classes.errorCard, className)}>
      {children}
    </div>
  );
}

export default AppErrorCard;
