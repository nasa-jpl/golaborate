"""bmcserver  enables http communication with the BMC commercial electronics."""

from flask import Flask, Blueprint


def create_app():
    """Create a new app instance, factory app pattern."""
    app = Flask(__name__)
    app.register_blueprint(bp)
    return app


bp = Blueprint('bmc', 'bmc')


@bp.route('/')
def bmcroot():
    """Serve the base route on GET and returns the last commanded DM state."""
    return "Hello world\n"


if __name__ == '__main__':
    from waitress import serve

    app = create_app()
    serve(app, listen='*:8000')
