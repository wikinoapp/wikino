# typed: strict

# DO NOT EDIT MANUALLY
# This file was pulled from https://raw.githubusercontent.com/Shopify/rbi-central/main.
# Please run `bin/tapioca annotations` to update it.

module Faraday
  class << self
    sig do
      params(
        url: T.untyped,
        options: T::Hash[Symbol, T.untyped],
        block: T.nilable(T.proc.params(connection: Faraday::Connection).void)
      ).returns(Faraday::Connection)
    end
    def new(url = nil, options = {}, &block); end
  end
end

class Faraday::Response
  sig { returns(T::Boolean) }
  def success?; end
end