import React, {PropsWithChildren} from "react";
import makeStyles from "@material-ui/core/styles/makeStyles";
import ButtonBase from "@material-ui/core/ButtonBase";
import {ButtonProps, createStyles, Theme} from '@material-ui/core';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    root: {
      position: "relative",
      border: "1px solid",
      borderColor: theme.palette.primary.main,
      borderRadius: "50%",
      flexShrink: 0,
      height: 36,
      width: 36,
      lineHeight: "34px",
      paddingLeft: 0,
      paddingRight: 0,
      paddingTop: 0,
      paddingBottom: 0,
      fontSize: "1rem",
      color: theme.palette.primary.main,
    },
    disabled: {
      borderColor: theme.palette.text.disabled,
      color: theme.palette.text.secondary,
    },
  })
);

function AppRoundButton(
  {children, disabled, className, ...props}: PropsWithChildren<{ className?: string }> & ButtonProps
) {
  const classes = useStyles();
  return (
    <ButtonBase
      {...props}
      className={
        classes.root +
        (disabled ? " " + classes.disabled : "") +
        (className ? " " + className : "")
      }
    >
      {children}
    </ButtonBase>
  );
}

export default AppRoundButton;
