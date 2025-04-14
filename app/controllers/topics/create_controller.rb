# typed: true
# frozen_string_literal: true

module Topics
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      space_viewer = Current.viewer!.space_viewer!(space:)

      unless space_viewer.can_create_topic?
        return render_404
      end

      form = NewTopicForm.new(form_params.merge(space:))

      if form.invalid?
        return render(
          Topics::NewView.new(
            current_user_entity: Current.viewer!.user_entity,
            space_entity: space.to_entity(space_viewer:),
            form:
          ),
          status: :unprocessable_entity
        )
      end

      result = CreateTopicService.new.call(
        space_member: T.let(space_viewer, SpaceMemberRecord),
        name: form.name.not_nil!,
        description: form.description.not_nil!,
        visibility: form.visibility.not_nil!
      )

      flash[:notice] = t("messages.topics.created")
      redirect_to topic_path(space.identifier, result.topic.number)
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
