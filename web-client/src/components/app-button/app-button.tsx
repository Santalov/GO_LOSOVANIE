import React from "react";
import ButtonBase from "@material-ui/core/ButtonBase";
import { makeStyles } from "@material-ui/core/styles";
import withTheme from "@material-ui/core/styles/withTheme";

function AppButtonRaw(props) {
  const classes = makeStyles({
    root: {
      position: "relative",
      border: "2px solid",
      borderColor: props.theme.palette.primary.main,
      borderRadius: 6,
      flexShrink: 0,
      height: 40,
      lineHeight: "36px",
      paddingLeft: 30,
      paddingRight: 30,
      paddingTop: 0,
      paddingBottom: 0,
      fontSize: "1rem",
      color: props.theme.palette.primary.main,
    },
    disabled: {
      borderColor: props.theme.palette.text.disabled,
      color: props.theme.palette.text.secondary,
    },
  })();
  return (
    <ButtonBase
      {...props}
      className={
        classes.root +
        (props.disabled ? " " + classes.disabled : "") +
        (props.className ? " " + props.className : "")
      }
    >
      {props.children}
    </ButtonBase>
  );
}

const AppButton = withTheme(AppButtonRaw);

export default AppButton;
