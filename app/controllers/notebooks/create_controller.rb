# typed: true
# frozen_string_literal: true

module Notebooks
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @form = NewNotebookForm.new(form_params.merge(viewer: viewer!))

      if @form.invalid?
        return render("notebooks/new/call", status: :unprocessable_entity)
      end

      result = CreateNotebookUseCase.new.call(
        viewer: viewer!,
        name: @form.name.not_nil!,
        description: @form.description.not_nil!,
        visibility: @form.visibility.not_nil!
      )

      flash[:notice] = t("messages.notebook.created")
      redirect_to notebook_path(viewer!.space_identifier, result.notebook.number)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:new_notebook_form), ActionController::Parameters).permit(
        :name,
        :description,
        :visibility
      )
    end
  end
end
