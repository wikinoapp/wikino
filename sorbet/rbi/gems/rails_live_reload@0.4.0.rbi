# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for types exported from the `rails_live_reload` gem.
# Please instead update this file by running `bin/tapioca gem rails_live_reload`.


# source://rails_live_reload//lib/rails_live_reload/version.rb#1
module RailsLiveReload
  # source://rails_live_reload//lib/rails_live_reload.rb#15
  def watcher; end

  # source://rails_live_reload//lib/rails_live_reload.rb#15
  def watcher=(val); end

  private

  # source://rails_live_reload//lib/rails_live_reload.rb#34
  def server; end

  class << self
    # source://rails_live_reload//lib/rails_live_reload/config.rb#7
    def config; end

    # @yield [config]
    #
    # source://rails_live_reload//lib/rails_live_reload/config.rb#3
    def configure; end

    # @return [Boolean]
    #
    # source://rails_live_reload//lib/rails_live_reload/config.rb#15
    def enabled?; end

    # source://rails_live_reload//lib/rails_live_reload/config.rb#11
    def patterns; end

    # source://rails_live_reload//lib/rails_live_reload.rb#34
    def server; end

    # source://rails_live_reload//lib/rails_live_reload.rb#15
    def watcher; end

    # source://rails_live_reload//lib/rails_live_reload.rb#15
    def watcher=(val); end
  end
end

# source://rails_live_reload//lib/rails_live_reload/checker.rb#2
class RailsLiveReload::Checker
  class << self
    # source://rails_live_reload//lib/rails_live_reload/checker.rb#3
    def files; end

    # source://rails_live_reload//lib/rails_live_reload/checker.rb#7
    def files=(files); end

    # source://rails_live_reload//lib/rails_live_reload/checker.rb#11
    def scan(dt, rendered_files); end
  end
end

# source://rails_live_reload//lib/rails_live_reload/command.rb#2
class RailsLiveReload::Command
  # @return [Command] a new instance of Command
  #
  # source://rails_live_reload//lib/rails_live_reload/command.rb#5
  def initialize(params); end

  # source://rails_live_reload//lib/rails_live_reload/command.rb#10
  def changes; end

  # Returns the value of attribute dt.
  #
  # source://rails_live_reload//lib/rails_live_reload/command.rb#3
  def dt; end

  # Returns the value of attribute files.
  #
  # source://rails_live_reload//lib/rails_live_reload/command.rb#3
  def files; end

  # source://rails_live_reload//lib/rails_live_reload/command.rb#18
  def payload; end

  # @return [Boolean]
  #
  # source://rails_live_reload//lib/rails_live_reload/command.rb#14
  def reload?; end
end

# source://rails_live_reload//lib/rails_live_reload/config.rb#20
class RailsLiveReload::Config
  # @return [Config] a new instance of Config
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#24
  def initialize; end

  # Returns the value of attribute enabled.
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#22
  def enabled; end

  # Sets the attribute enabled
  #
  # @param value the value to set the attribute enabled to.
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#22
  def enabled=(_arg0); end

  # Returns the value of attribute files.
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#22
  def files; end

  # Sets the attribute files
  #
  # @param value the value to set the attribute files to.
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#22
  def files=(_arg0); end

  # Returns the value of attribute patterns.
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#21
  def patterns; end

  # source://rails_live_reload//lib/rails_live_reload/config.rb#38
  def root_path; end

  # source://rails_live_reload//lib/rails_live_reload/config.rb#51
  def socket_path; end

  # Returns the value of attribute url.
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#22
  def url; end

  # Sets the attribute url
  #
  # @param value the value to set the attribute url to.
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#22
  def url=(_arg0); end

  # source://rails_live_reload//lib/rails_live_reload/config.rb#42
  def watch(pattern, reload: T.unsafe(nil)); end

  # Returns the value of attribute watcher.
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#22
  def watcher; end

  # Sets the attribute watcher
  #
  # @param value the value to set the attribute watcher to.
  #
  # source://rails_live_reload//lib/rails_live_reload/config.rb#22
  def watcher=(_arg0); end
end

# source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#2
class RailsLiveReload::CurrentRequest
  # @return [CurrentRequest] a new instance of CurrentRequest
  #
  # source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#18
  def initialize(request_id); end

  # Returns the value of attribute data.
  #
  # source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#3
  def data; end

  # Sets the attribute data
  #
  # @param value the value to set the attribute data to.
  #
  # source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#3
  def data=(_arg0); end

  # Returns the value of attribute record.
  #
  # source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#3
  def record; end

  # Sets the attribute record
  #
  # @param value the value to set the attribute record to.
  #
  # source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#3
  def record=(_arg0); end

  # Returns the value of attribute request_id.
  #
  # source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#4
  def request_id; end

  class << self
    # source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#14
    def cleanup; end

    # source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#10
    def current; end

    # source://rails_live_reload//lib/rails_live_reload/thread/current_request.rb#6
    def init; end
  end
end

# source://rails_live_reload//lib/rails_live_reload.rb#18
RailsLiveReload::INTERNAL = T.let(T.unsafe(nil), Hash)

# source://rails_live_reload//lib/rails_live_reload/instrument/metrics_collector.rb#2
module RailsLiveReload::Instrument; end

# source://rails_live_reload//lib/rails_live_reload/instrument/metrics_collector.rb#3
class RailsLiveReload::Instrument::MetricsCollector
  # source://rails_live_reload//lib/rails_live_reload/instrument/metrics_collector.rb#4
  def call(event_name, started, finished, event_id, payload); end
end

# source://rails_live_reload//lib/rails_live_reload/middleware/base.rb#2
module RailsLiveReload::Middleware; end

# source://rails_live_reload//lib/rails_live_reload/middleware/base.rb#3
class RailsLiveReload::Middleware::Base
  # @return [Base] a new instance of Base
  #
  # source://rails_live_reload//lib/rails_live_reload/middleware/base.rb#4
  def initialize(app); end

  # source://rails_live_reload//lib/rails_live_reload/middleware/base.rb#8
  def call(env); end

  # source://rails_live_reload//lib/rails_live_reload/middleware/base.rb#12
  def call!(env); end

  private

  # @return [Boolean]
  #
  # source://rails_live_reload//lib/rails_live_reload/middleware/base.rb#60
  def html?(headers); end

  # source://rails_live_reload//lib/rails_live_reload/middleware/base.rb#31
  def inject_rails_live_reload(request, status, headers, body); end

  # source://rails_live_reload//lib/rails_live_reload/middleware/base.rb#44
  def make_new_response(body, nonce); end
end

# source://rails_live_reload//lib/rails_live_reload/engine.rb#2
class RailsLiveReload::Railtie < ::Rails::Engine
  # @return [Boolean]
  #
  # source://rails_live_reload//lib/rails_live_reload/engine.rb#3
  def enabled?; end

  class << self
    # source://activesupport/7.1.5.1/lib/active_support/callbacks.rb#70
    def __callbacks; end
  end
end

# source://rails_live_reload//lib/rails_live_reload/server/connections.rb#2
module RailsLiveReload::Server; end

# This class is based on ActionCable
# https://github.com/rails/rails/blob/v7.0.3/actioncable/lib/action_cable/server/base.rb
#
# source://rails_live_reload//lib/rails_live_reload/server/base.rb#12
class RailsLiveReload::Server::Base
  include ::RailsLiveReload::Server::Connections

  # @return [Base] a new instance of Base
  #
  # source://rails_live_reload//lib/rails_live_reload/server/base.rb#23
  def initialize; end

  # Called by Rack to set up the server.
  #
  # source://rails_live_reload//lib/rails_live_reload/server/base.rb#29
  def call(env); end

  # source://rails_live_reload//lib/rails_live_reload/server/base.rb#44
  def client_javascript; end

  # source://rails_live_reload//lib/rails_live_reload/server/base.rb#48
  def event_loop; end

  # Returns the value of attribute mutex.
  #
  # source://rails_live_reload//lib/rails_live_reload/server/base.rb#15
  def mutex; end

  # source://rails_live_reload//lib/rails_live_reload/server/base.rb#17
  def reload_all; end

  # source://rails_live_reload//lib/rails_live_reload/server/base.rb#52
  def setup_socket; end
end

# This class is strongly based on ActionCable
# https://github.com/rails/rails/blob/v7.0.3/actioncable/lib/action_cable/server/connections.rb
#
# source://rails_live_reload//lib/rails_live_reload/server/connections.rb#5
module RailsLiveReload::Server::Connections
  # source://rails_live_reload//lib/rails_live_reload/server/connections.rb#12
  def add_connection(connection); end

  # source://rails_live_reload//lib/rails_live_reload/server/connections.rb#8
  def connections; end

  # source://rails_live_reload//lib/rails_live_reload/server/connections.rb#16
  def remove_connection(connection); end

  # source://rails_live_reload//lib/rails_live_reload/server/connections.rb#20
  def setup_heartbeat_timer; end
end

# source://rails_live_reload//lib/rails_live_reload/server/connections.rb#6
RailsLiveReload::Server::Connections::BEAT_INTERVAL = T.let(T.unsafe(nil), Integer)

# source://rails_live_reload//lib/rails_live_reload/version.rb#2
RailsLiveReload::VERSION = T.let(T.unsafe(nil), String)

# source://rails_live_reload//lib/rails_live_reload/watcher.rb#2
class RailsLiveReload::Watcher
  # @return [Watcher] a new instance of Watcher
  #
  # source://rails_live_reload//lib/rails_live_reload/watcher.rb#14
  def initialize; end

  # source://rails_live_reload//lib/rails_live_reload/watcher.rb#41
  def build_tree; end

  # Returns the value of attribute files.
  #
  # source://rails_live_reload//lib/rails_live_reload/watcher.rb#3
  def files; end

  # source://rails_live_reload//lib/rails_live_reload/watcher.rb#47
  def reload_all; end

  # source://rails_live_reload//lib/rails_live_reload/watcher.rb#5
  def root; end

  # Returns the value of attribute sockets.
  #
  # source://rails_live_reload//lib/rails_live_reload/watcher.rb#3
  def sockets; end

  # source://rails_live_reload//lib/rails_live_reload/watcher.rb#28
  def start_listener; end

  # source://rails_live_reload//lib/rails_live_reload/watcher.rb#58
  def start_socket; end

  class << self
    # source://rails_live_reload//lib/rails_live_reload/watcher.rb#9
    def init; end
  end
end

# source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#4
module RailsLiveReload::WebSocket; end

# This class is strongly based on ActionCable
# https://github.com/rails/rails/blob/v7.0.3/actioncable/lib/action_cable/connection/base.rb
#
# source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#5
class RailsLiveReload::WebSocket::Base
  # @return [Base] a new instance of Base
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#11
  def initialize(server, request); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#59
  def beat; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#51
  def close(reason: T.unsafe(nil)); end

  # Returns the value of attribute dt.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#7
  def dt; end

  # Returns the value of attribute env.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#6
  def env; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#9
  def event_loop(*_arg0, **_arg1, &_arg2); end

  # Returns the value of attribute files.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#7
  def files; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#32
  def handle_channel_command(payload); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#79
  def on_close(reason, code); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#75
  def on_error(message); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#71
  def on_message(message); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#63
  def on_open; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#18
  def process; end

  # Returns the value of attribute protocol.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#6
  def protocol; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#26
  def receive(websocket_message); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#41
  def reload; end

  # Returns the value of attribute request.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#6
  def request; end

  # Returns the value of attribute server.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#6
  def server; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#47
  def transmit(cable_message); end

  private

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#97
  def decode(message); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#93
  def encode(message); end

  # Returns the value of attribute message_buffer.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#86
  def message_buffer; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#109
  def respond_to_invalid_request; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#105
  def respond_to_successful_request; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#101
  def send_welcome_message; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#88
  def setup(options); end

  # Returns the value of attribute websocket.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/base.rb#85
  def websocket; end
end

# This class is basically copied from ActionCable
# https://github.com/rails/rails/blob/v7.0.3/actioncable/lib/action_cable/connection/client_socket.rb
#
# source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#7
class RailsLiveReload::WebSocket::ClientSocket
  # @return [ClientSocket] a new instance of ClientSocket
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#30
  def initialize(env, event_target, event_loop, protocols); end

  # @return [Boolean]
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#107
  def alive?; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#103
  def client_gone; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#85
  def close(code = T.unsafe(nil), reason = T.unsafe(nil)); end

  # Returns the value of attribute env.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#28
  def env; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#99
  def parse(data); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#111
  def protocol; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#64
  def rack_response; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#52
  def start_driver; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#75
  def transmit(message); end

  # Returns the value of attribute url.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#28
  def url; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#69
  def write(data); end

  private

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#136
  def begin_close(reason, code); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#130
  def emit_error(message); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#145
  def finalize_close; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#117
  def open; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#124
  def receive_message(data); end

  class << self
    # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#8
    def determine_url(env); end

    # @return [Boolean]
    #
    # source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#13
    def secure_request?(env); end
  end
end

# source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#26
RailsLiveReload::WebSocket::ClientSocket::CLOSED = T.let(T.unsafe(nil), Integer)

# source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#25
RailsLiveReload::WebSocket::ClientSocket::CLOSING = T.let(T.unsafe(nil), Integer)

# source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#23
RailsLiveReload::WebSocket::ClientSocket::CONNECTING = T.let(T.unsafe(nil), Integer)

# source://rails_live_reload//lib/rails_live_reload/web_socket/client_socket.rb#24
RailsLiveReload::WebSocket::ClientSocket::OPEN = T.let(T.unsafe(nil), Integer)

# This class is basically copied from ActionCable
# https://github.com/rails/rails/blob/v7.0.3/actioncable/lib/action_cable/connection/stream_event_loop.rb
#
# source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#7
class RailsLiveReload::WebSocket::EventLoop
  # @return [EventLoop] a new instance of EventLoop
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#8
  def initialize; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#27
  def attach(io, stream); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#35
  def detach(io, stream); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#20
  def post(task = T.unsafe(nil), &block); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#53
  def stop; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#16
  def timer(interval, &block); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#44
  def writes_pending(io); end

  private

  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#84
  def run; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#60
  def spawn; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/event_loop.rb#80
  def wakeup; end
end

# This class is basically copied from ActionCable
# https://github.com/rails/rails/blob/v7.0.3/actioncable/lib/action_cable/connection/message_buffer.rb
#
# source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#5
class RailsLiveReload::WebSocket::MessageBuffer
  # @return [MessageBuffer] a new instance of MessageBuffer
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#6
  def initialize(connection); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#11
  def append(message); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#27
  def process!; end

  # @return [Boolean]
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#23
  def processing?; end

  private

  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#44
  def buffer(message); end

  # Returns the value of attribute buffered_messages.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#34
  def buffered_messages; end

  # Returns the value of attribute connection.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#34
  def connection; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#40
  def receive(message); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#48
  def receive_buffered_messages; end

  # @return [Boolean]
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/message_buffer.rb#36
  def valid?(message); end
end

# This class is basically copied from ActionCable
# https://github.com/rails/rails/blob/v7.0.3/actioncable/lib/action_cable/connection/stream.rb
#
# source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#5
class RailsLiveReload::WebSocket::Stream
  # @return [Stream] a new instance of Stream
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#6
  def initialize(event_loop, socket); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#22
  def close; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#18
  def each(&callback); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#65
  def flush_write_buffer; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#91
  def hijack_rack_socket; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#87
  def receive(data); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#27
  def shutdown; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#31
  def write(data); end

  private

  # source://rails_live_reload//lib/rails_live_reload/web_socket/stream.rb#102
  def clean_rack_hijack; end
end

# This class is basically copied from ActionCable
# https://github.com/rails/rails/blob/v7.0.3/actioncable/lib/action_cable/connection/web_socket.rb
#
# source://rails_live_reload//lib/rails_live_reload/web_socket/wrapper.rb#7
class RailsLiveReload::WebSocket::Wrapper
  # @return [Wrapper] a new instance of Wrapper
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/wrapper.rb#10
  def initialize(env, event_target, event_loop, protocols: T.unsafe(nil)); end

  # @return [Boolean]
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/wrapper.rb#18
  def alive?; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/wrapper.rb#8
  def close(*_arg0, **_arg1, &_arg2); end

  # @return [Boolean]
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/wrapper.rb#14
  def possible?; end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/wrapper.rb#8
  def protocol(*_arg0, **_arg1, &_arg2); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/wrapper.rb#8
  def rack_response(*_arg0, **_arg1, &_arg2); end

  # source://rails_live_reload//lib/rails_live_reload/web_socket/wrapper.rb#8
  def transmit(*_arg0, **_arg1, &_arg2); end

  private

  # Returns the value of attribute websocket.
  #
  # source://rails_live_reload//lib/rails_live_reload/web_socket/wrapper.rb#24
  def websocket; end
end
