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
      form = SpaceForm::Creation.new(form_params)

      if form.invalid?
        return render_new_view(form:)
      end

      result = SpaceService::Create.new.call(
        user_record: current_user_record!,
        identifier: form.identifier.not_nil!,
        name: form.name.not_nil!
      )

      flash[:notice] = t("messages.spaces.created")
      redirect_to space_path(result.space_record.identifier)
    rescue ApplicationService::RecordNotUniqueError => e
      form.not_nil!.errors.add(e.attribute, e.message)
      render_new_view(form:)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:space_form_creation), ActionController::Parameters).permit(
        :identifier,
        :name
      )
    end

    private def render_new_view(form:)
      render_component(
        Spaces::NewView.new(
          current_user: current_user!,
          form:
        ),
        status: :unprocessable_entity
      )
    end
  end
end
