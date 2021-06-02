import React, {PropsWithChildren} from "react";
import {createStyles, makeStyles, Theme} from '@material-ui/core';
import classNames from 'classnames';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    main: {
      marginBottom: theme.spacing(0.5),
    },
    label: {
      color: theme.palette.text.secondary,
      fontSize: "0.85rem",
    },
    content: {
      color: theme.palette.text.primary,
      fontSize: "1rem",
      wordWrap: "break-word",
    },
    labelLarge: {
      fontSize: '1rem',
    },
    contentLarge: {
      fontSize: '1.2rem'
    }
  })
);

function AppInfoLine(
  {label, children, large}: PropsWithChildren<{ label: string, large?: boolean }>) {
  const classes = useStyles();
  return (
    <div className={classes.main}>
      <div
        className={classNames(classes.label, {[classes.labelLarge]: large})}
      >
        {label}
      </div>
      <div
        className={classNames(classes.content, {[classes.contentLarge]: large})}
      >
        {children}
      </div>
    </div>
  );
}

export default AppInfoLine;
