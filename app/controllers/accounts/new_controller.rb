# typed: true
# frozen_string_literal: true

module Accounts
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::EmailConfirmationFindable

    around_action :set_locale
    before_action :require_no_authentication
    before_action :require_succeeded_email_confirmation

    sig { returns(T.untyped) }
    def call
      form = AccountForm::Creation.new(email: @email_confirmation.not_nil!.email.not_nil!)

      render Accounts::NewView.new(form:)
    end
  end
end
