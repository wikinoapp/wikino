# typed: true
# frozen_string_literal: true

module Topics
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceAware

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = current_space_record
      space_policy = space_policy_for(space_record:)

      unless space_policy.can_create_topic?
        return render_404
      end

      space_member_record = current_space_member_record!(space_record:)

      form = Topics::CreationForm.new(form_params.merge(space_record:))

      if form.invalid?
        space = SpaceRepository.new.to_model(space_record:)

        return render_component(
          Topics::NewView.new(
            current_user: current_user!,
            space:,
            form:
          ),
          status: :unprocessable_entity
        )
      end

      result = Topics::CreateService.new.call(
        space_member_record: space_member_record.not_nil!,
        name: form.name.not_nil!,
        description: form.description.not_nil!,
        visibility: form.visibility.not_nil!
      )

      flash[:notice] = t("messages.topics.created")
      redirect_to topic_path(space_record.identifier, result.topic_record.number)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:topics_creation_form), ActionController::Parameters).permit(
        :name,
        :description,
        :visibility
      )
    end
  end
end
