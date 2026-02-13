# typed: strict
# frozen_string_literal: true

module Test
  module SignIn
    class CreateController < ApplicationController
      include ControllerConcerns::Authenticatable

      sig { returns(T.untyped) }
      def call
        user_record = UserRecord.find(params[:user_id])
        user_session_record = user_record.user_session_records.start!(
          ip_address: "127.0.0.1",
          user_agent: "SystemSpec"
        )
        sign_in(user_session_record)
        redirect_to "/home"
      end
    end
  end
end
