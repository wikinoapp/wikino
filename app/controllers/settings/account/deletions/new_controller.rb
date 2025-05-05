# typed: strict
# frozen_string_literal: true

module Settings
  module Account
    module Deletions
      class NewController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          render Settings::Account::Deletions::NewView.new(
            current_user: current_user!
          )
        end
      end
    end
  end
end
