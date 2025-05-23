# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for types exported from the `resend` gem.
# Please instead update this file by running `bin/tapioca gem resend`.


# Main Resend module
#
# source://resend//lib/resend/version.rb#3
module Resend
  class << self
    # Returns the value of attribute api_key.
    #
    # source://resend//lib/resend.rb#28
    def api_key; end

    # Sets the attribute api_key
    #
    # @param value the value to set the attribute api_key to.
    #
    # source://resend//lib/resend.rb#28
    def api_key=(_arg0); end

    # @yield [_self]
    # @yieldparam _self [Resend] the object that the method was called on
    #
    # source://resend//lib/resend.rb#30
    def config; end

    # @yield [_self]
    # @yieldparam _self [Resend] the object that the method was called on
    #
    # source://resend//lib/resend.rb#30
    def configure; end
  end
end

# api keys api wrapper
#
# source://resend//lib/resend/api_keys.rb#5
module Resend::ApiKeys
  class << self
    # https://resend.com/docs/api-reference/api-keys/create-api-key
    #
    # source://resend//lib/resend/api_keys.rb#8
    def create(params); end

    # https://resend.com/docs/api-reference/api-keys/list-api-keys
    #
    # source://resend//lib/resend/api_keys.rb#14
    def list; end

    # https://resend.com/docs/api-reference/api-keys/delete-api-key
    #
    # source://resend//lib/resend/api_keys.rb#20
    def remove(api_key_id = T.unsafe(nil)); end
  end
end

# Audiences api wrapper
#
# source://resend//lib/resend/audiences.rb#5
module Resend::Audiences
  class << self
    # https://resend.com/docs/api-reference/audiences/create-audience
    #
    # source://resend//lib/resend/audiences.rb#8
    def create(params); end

    # https://resend.com/docs/api-reference/audiences/get-audience
    #
    # source://resend//lib/resend/audiences.rb#14
    def get(audience_id = T.unsafe(nil)); end

    # https://resend.com/docs/api-reference/audiences/list-audiences
    #
    # source://resend//lib/resend/audiences.rb#20
    def list; end

    # https://resend.com/docs/api-reference/audiences/delete-audience
    #
    # source://resend//lib/resend/audiences.rb#26
    def remove(audience_id = T.unsafe(nil)); end
  end
end

# Module responsible for wrapping Batch email sending API
#
# source://resend//lib/resend/batch.rb#5
module Resend::Batch
  class << self
    # https://resend.com/docs/api-reference/emails/send-batch-emails
    #
    # source://resend//lib/resend/batch.rb#8
    def send(params = T.unsafe(nil), options: T.unsafe(nil)); end
  end
end

# broadcasts api wrapper
#
# source://resend//lib/resend/broadcasts.rb#5
module Resend::Broadcasts
  class << self
    # https://resend.com/docs/api-reference/broadcasts/create-broadcast
    #
    # source://resend//lib/resend/broadcasts.rb#8
    def create(params = T.unsafe(nil)); end

    # https://resend.com/docs/api-reference/broadcasts/get-broadcast
    #
    # source://resend//lib/resend/broadcasts.rb#38
    def get(broadcast_id = T.unsafe(nil)); end

    # https://resend.com/docs/api-reference/broadcasts/list-broadcasts
    #
    # source://resend//lib/resend/broadcasts.rb#26
    def list; end

    # https://resend.com/docs/api-reference/broadcasts/delete-broadcast
    #
    # source://resend//lib/resend/broadcasts.rb#32
    def remove(broadcast_id = T.unsafe(nil)); end

    # https://resend.com/docs/api-reference/broadcasts/send-broadcast
    #
    # source://resend//lib/resend/broadcasts.rb#20
    def send(params = T.unsafe(nil)); end

    # https://resend.com/docs/api-reference/broadcasts/update-broadcast
    #
    # source://resend//lib/resend/broadcasts.rb#14
    def update(params = T.unsafe(nil)); end
  end
end

# Client class.
#
# source://resend//lib/resend/client.rb#8
class Resend::Client
  include ::Resend::Emails

  # @raise [ArgumentError]
  # @return [Client] a new instance of Client
  #
  # source://resend//lib/resend/client.rb#13
  def initialize(api_key); end

  # Returns the value of attribute api_key.
  #
  # source://resend//lib/resend/client.rb#11
  def api_key; end
end

# Contacts api wrapper
#
# source://resend//lib/resend/contacts.rb#5
module Resend::Contacts
  class << self
    # https://resend.com/docs/api-reference/contacts/create-contact
    #
    # source://resend//lib/resend/contacts.rb#8
    def create(params); end

    # Retrieves a contact from an audience
    #
    # https://resend.com/docs/api-reference/contacts/get-contact
    #
    # @param audience_id [String] the audience id
    # @param id [String] either the contact id or contact's email
    #
    # source://resend//lib/resend/contacts.rb#20
    def get(audience_id, id); end

    # List contacts in an audience
    #
    # https://resend.com/docs/api-reference/contacts/list-contacts
    #
    # @param audience_id [String] the audience id
    #
    # source://resend//lib/resend/contacts.rb#30
    def list(audience_id); end

    # Remove a contact from an audience
    #
    # see also: https://resend.com/docs/api-reference/contacts/delete-contact
    #
    # @param audience_id [String] the audience id
    # @param contact_id [String] either the contact id or contact email
    #
    # source://resend//lib/resend/contacts.rb#42
    def remove(audience_id, contact_id); end

    # Update a contact
    #
    # https://resend.com/docs/api-reference/contacts/update-contact
    #
    # @param params [Hash] the contact params
    # @raise [ArgumentError]
    #
    # source://resend//lib/resend/contacts.rb#52
    def update(params); end
  end
end

# domains api wrapper
#
# source://resend//lib/resend/domains.rb#5
module Resend::Domains
  class << self
    # https://resend.com/docs/api-reference/domains/create-domain
    #
    # source://resend//lib/resend/domains.rb#8
    def create(params); end

    # https://resend.com/docs/api-reference/domains/get-domain
    #
    # source://resend//lib/resend/domains.rb#20
    def get(domain_id = T.unsafe(nil)); end

    # https://resend.com/docs/api-reference/api-keys/list-api-keys
    #
    # source://resend//lib/resend/domains.rb#26
    def list; end

    # https://resend.com/docs/api-reference/domains/delete-domain
    #
    # source://resend//lib/resend/domains.rb#32
    def remove(domain_id = T.unsafe(nil)); end

    # https://resend.com/docs/api-reference/domains/update-domain
    #
    # source://resend//lib/resend/domains.rb#14
    def update(params); end

    # https://resend.com/docs/api-reference/domains/verify-domain
    #
    # source://resend//lib/resend/domains.rb#38
    def verify(domain_id = T.unsafe(nil)); end
  end
end

# Module responsible for wrapping email sending API
#
# source://resend//lib/resend/emails.rb#5
module Resend::Emails
  # This method is kept here for backwards compatibility
  # Use Resend::Emails.send instead.
  #
  # source://resend//lib/resend/emails.rb#38
  def send_email(params); end

  class << self
    # Cancel a scheduled email.
    # see more: https://resend.com/docs/api-reference/emails/cancel-email
    #
    # source://resend//lib/resend/emails.rb#30
    def cancel(email_id = T.unsafe(nil)); end

    # Retrieve a single email.
    # see more: https://resend.com/docs/api-reference/emails/retrieve-email
    #
    # source://resend//lib/resend/emails.rb#16
    def get(email_id = T.unsafe(nil)); end

    # Sends or schedules an email.
    # see more: https://resend.com/docs/api-reference/emails/send-email
    #
    # source://resend//lib/resend/emails.rb#9
    def send(params, options: T.unsafe(nil)); end

    # Update a scheduled email.
    # see more: https://resend.com/docs/api-reference/emails/update-email
    #
    # source://resend//lib/resend/emails.rb#23
    def update(params); end
  end
end

# Errors wrapper class
# For more info: https://resend.com/docs/api-reference/error-codes
#
# source://resend//lib/resend/errors.rb#6
class Resend::Error < ::StandardError
  # @return [Error] a new instance of Error
  #
  # source://resend//lib/resend/errors.rb#34
  def initialize(msg, code = T.unsafe(nil)); end
end

# 4xx HTTP status code
#
# source://resend//lib/resend/errors.rb#8
class Resend::Error::ClientError < ::Resend::Error; end

# source://resend//lib/resend/errors.rb#25
Resend::Error::ERRORS = T.let(T.unsafe(nil), Hash)

# code 500
#
# source://resend//lib/resend/errors.rb#14
class Resend::Error::InternalServerError < ::Resend::Error::ServerError; end

# code 422
#
# source://resend//lib/resend/errors.rb#17
class Resend::Error::InvalidRequestError < ::Resend::Error::ServerError; end

# code 404
#
# source://resend//lib/resend/errors.rb#23
class Resend::Error::NotFoundError < ::Resend::Error::ServerError; end

# code 429
#
# source://resend//lib/resend/errors.rb#20
class Resend::Error::RateLimitExceededError < ::Resend::Error::ServerError; end

# 5xx HTTP status code
#
# source://resend//lib/resend/errors.rb#11
class Resend::Error::ServerError < ::Resend::Error; end

# Mailer class used by railtie
#
# source://resend//lib/resend/mailer.rb#7
class Resend::Mailer
  # @raise [Resend::Error]
  # @return [Mailer] a new instance of Mailer
  #
  # source://resend//lib/resend/mailer.rb#23
  def initialize(config); end

  # Builds the payload for sending
  #
  # @param Mail mail rails mail object
  # @return Hash hash with all Resend params
  #
  # source://resend//lib/resend/mailer.rb#53
  def build_resend_params(mail); end

  # Remove nils from header values
  #
  # source://resend//lib/resend/mailer.rb#115
  def cleanup_headers(headers); end

  # Returns the value of attribute config.
  #
  # source://resend//lib/resend/mailer.rb#8
  def config; end

  # Sets the attribute config
  #
  # @param value the value to set the attribute config to.
  #
  # source://resend//lib/resend/mailer.rb#8
  def config=(_arg0); end

  # Overwritten deliver! method
  #
  # @param Mail mail
  # @return Object resend response
  #
  # source://resend//lib/resend/mailer.rb#38
  def deliver!(mail); end

  # Add cc, bcc, reply_to fields
  #
  # @param Mail mail Rails Mail Object
  # @return Hash hash containing cc/bcc/reply_to attrs
  #
  # source://resend//lib/resend/mailer.rb#165
  def get_addons(mail); end

  # Handle attachments when present
  #
  # @return Array attachments array
  #
  # source://resend//lib/resend/mailer.rb#215
  def get_attachments(mail); end

  # Gets the body of the email
  #
  # @param Mail mail Rails Mail Object
  # @return Hash hash containing html/text or both attrs
  #
  # source://resend//lib/resend/mailer.rb#180
  def get_contents(mail); end

  # Properly gets the `from` attr
  #
  # @param Mail input object
  # @return String `from` string
  #
  # source://resend//lib/resend/mailer.rb#201
  def get_from(input); end

  # Add custom headers fields.
  #
  # Both ways are supported:
  #
  #   1. Through the `#mail()` method ie:
  #     mail(headers: { "X-Custom-Header" => "value" })
  #
  #   2. Through the Rails `#headers` method ie:
  #     headers["X-Custom-Header"] = "value"
  #
  #
  # setting the header values through the `#mail` method will overwrite values set
  # through the `#headers` method using the same key.
  #
  # @param Mail mail Rails Mail object
  # @return Hash hash with headers param
  #
  # source://resend//lib/resend/mailer.rb#86
  def get_headers(mail); end

  # Adds additional options fields.
  # Currently supports only :idempotency_key
  #
  # @param Mail mail Rails Mail object
  # @return Hash hash with headers param
  #
  # source://resend//lib/resend/mailer.rb#105
  def get_options(mail); end

  # Add tags fields
  #
  # @param Mail mail Rails Mail object
  # @return Hash hash with tags param
  #
  # source://resend//lib/resend/mailer.rb#152
  def get_tags(mail); end

  # Gets the values of the headers that are set through the `#headers` method
  #
  # @param Mail mail Rails Mail object
  # @return Hash hash with headers values
  #
  # source://resend//lib/resend/mailer.rb#136
  def headers_values(mail); end

  # Gets the values of the headers that are set through the `#mail` method
  #
  # @param Mail mail Rails Mail object
  # @return Hash hash with mail headers values
  #
  # source://resend//lib/resend/mailer.rb#123
  def mail_headers_values(mail); end

  # Returns the value of attribute settings.
  #
  # source://resend//lib/resend/mailer.rb#8
  def settings; end

  # Sets the attribute settings
  #
  # @param value the value to set the attribute settings to.
  #
  # source://resend//lib/resend/mailer.rb#8
  def settings=(_arg0); end

  # Get all headers that are not ignored
  #
  # @param Mail mail
  # @return Array headers
  #
  # source://resend//lib/resend/mailer.rb#234
  def unignored_headers(mail); end
end

# These are set as `headers` by the Rails API, but these will be filtered out
# when constructing the Resend API payload, since they're are sent as post params.
# https://resend.com/docs/api-reference/emails/send-email
#
# source://resend//lib/resend/mailer.rb#13
Resend::Mailer::IGNORED_HEADERS = T.let(T.unsafe(nil), Array)

# source://resend//lib/resend/mailer.rb#21
Resend::Mailer::SUPPORTED_OPTIONS = T.let(T.unsafe(nil), Array)

# Main railtime class
#
# source://resend//lib/resend/railtie.rb#8
class Resend::Railtie < ::Rails::Railtie; end

# This class is responsible for making the appropriate HTTP calls
# and raising the specific errors based on the response.
#
# source://resend//lib/resend/request.rb#6
class Resend::Request
  # @return [Request] a new instance of Request
  #
  # source://resend//lib/resend/request.rb#11
  def initialize(path = T.unsafe(nil), body = T.unsafe(nil), verb = T.unsafe(nil), options: T.unsafe(nil)); end

  # Returns the value of attribute body.
  #
  # source://resend//lib/resend/request.rb#9
  def body; end

  # Sets the attribute body
  #
  # @param value the value to set the attribute body to.
  #
  # source://resend//lib/resend/request.rb#9
  def body=(_arg0); end

  # @raise [error]
  #
  # source://resend//lib/resend/request.rb#46
  def handle_error!(resp); end

  # Returns the value of attribute options.
  #
  # source://resend//lib/resend/request.rb#9
  def options; end

  # Sets the attribute options
  #
  # @param value the value to set the attribute options to.
  #
  # source://resend//lib/resend/request.rb#9
  def options=(_arg0); end

  # Performs the HTTP call
  #
  # source://resend//lib/resend/request.rb#31
  def perform; end

  # Returns the value of attribute verb.
  #
  # source://resend//lib/resend/request.rb#9
  def verb; end

  # Sets the attribute verb
  #
  # @param value the value to set the attribute verb to.
  #
  # source://resend//lib/resend/request.rb#9
  def verb=(_arg0); end

  private

  # source://resend//lib/resend/request.rb#69
  def check_json!(resp); end

  # source://resend//lib/resend/request.rb#60
  def set_idempotency_key; end
end

# source://resend//lib/resend/request.rb#7
Resend::Request::BASE_URL = T.let(T.unsafe(nil), String)

# source://resend//lib/resend/version.rb#4
Resend::VERSION = T.let(T.unsafe(nil), String)
