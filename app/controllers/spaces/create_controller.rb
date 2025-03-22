# typed: true
# frozen_string_literal: true

module Spaces
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      form = NewSpaceForm.new(form_params)

      if form.invalid?
        return render(
          Spaces::NewView.new(
            current_user_entity: Current.viewer!.user_entity,
            form:
          ),
          status: :unprocessable_entity
        )
      end

      result = CreateSpaceService.new.call(
        user: T.let(Current.viewer!, User),
        identifier: form.identifier.not_nil!,
        name: form.name.not_nil!
      )

      flash[:notice] = t("messages.spaces.created")
      redirect_to space_path(result.space.identifier)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:new_space_form), ActionController::Parameters).permit(
        :identifier,
        :name
      )
    end
  end
end
