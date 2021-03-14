# frozen_string_literal: true

module Types
  module Objects
    class AccessTokenType < Types::Objects::Base
      field :token, String, null: false
    end
  end
end
