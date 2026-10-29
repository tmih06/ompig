import ompig_sdk


def new_extension() -> ompig_sdk.Extension:
    ext = ompig_sdk.Extension("pywidget")
    ext.on_event(
        "session_start",
        lambda ctx, _event: ctx.set_widget("sdk-wide", ["WIDGET-START " + "W" * 150 + " tail words wrap here", "short entry"]),
    )
    return ext
