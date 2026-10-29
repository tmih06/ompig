from pathlib import Path
import ompig_sdk


def new_extension() -> ompig_sdk.Extension:
    order = Path("d77-order")
    if order.read_text() != "A":
        raise RuntimeError("B admitted before A")
    with order.open("a") as stream:
        stream.write("B")
    return ompig_sdk.Extension("middle")
