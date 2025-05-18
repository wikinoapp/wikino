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
          form = AccountForm::DestroyConfirmation.new

          active_space_records = current_user_record!.active_space_records.order(:identifier)
          active_spaces = SpaceRepository.new.to_models(space_records: active_space_records)

          render_component Settings::Account::Deletions::NewView.new(
            current_user: current_user!,
            form:,
            active_spaces:
          )
        end
      end
    end
  end
end
