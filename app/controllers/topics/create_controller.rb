# typed: true
# frozen_string_literal: true

module Topics
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @form = NewTopicForm.new(form_params)

      if @form.invalid?
        return render("topics/new/call", status: :unprocessable_entity)
      end

      result = CreateTopicUseCase.new.call(
        name: @form.name.not_nil!,
        description: @form.description.not_nil!,
        visibility: @form.visibility.not_nil!
      )

      flash[:notice] = t("messages.topic.created")
      redirect_to topic_path(Current.space!.identifier, result.topic.number)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:new_topic_form), ActionController::Parameters).permit(
        :name,
        :description,
        :visibility
      )
    end
  end
end
