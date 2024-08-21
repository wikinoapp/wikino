# typed: true
# frozen_string_literal: true

module Lists
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @form = NewListForm.new(form_params.merge(viewer: viewer!))

      if @form.invalid?
        return render("lists/new/call", status: :unprocessable_entity)
      end

      result = CreateListUseCase.new.call(
        viewer: viewer!,
        identifier: @form.identifier.not_nil!,
        visibility: @form.visibility.not_nil!,
        name: @form.name.not_nil!,
        description: @form.description.not_nil!
      )

      flash[:notice] = t("messages.list.created")
      redirect_to list_path(viewer!.space_identifier, result.list.identifier)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:new_list_form), ActionController::Parameters).permit(
        :identifier,
        :visibility,
        :name,
        :description
      )
    end
  end
end
