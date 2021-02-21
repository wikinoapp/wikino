# frozen_string_literal: true

module Types
  module Enums
    class EmailConfirmationEvent < Types::Enums::Base
      value "SIGN_IN", "ログインするとき"
      value "SIGN_UP", "ユーザ登録するとき"
    end
  end
end
