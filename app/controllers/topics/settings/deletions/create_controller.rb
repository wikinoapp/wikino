# typed: true
# frozen_string_literal: true

module Topics
  module Settings
    module Deletions
      class CreateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_member_record = current_user_record!.space_member_record(space_record:)
          space_member_policy = SpaceMemberPolicy.new(
            user_record: current_user_record!,
            space_member_record:
          )
          topic_record = space_member_policy.showable_topics(space_record:).find_by!(
            number: params[:topic_number]
          )

          unless space_member_policy.can_update_topic?(topic_record:)
            return render_404
          end

          form = TopicForm::DestroyConfirmation.new(form_params.merge(topic_record:))

          if form.invalid?
            topic = TopicRepository.new.to_model(topic_record:)

            return render(
              Topics::Settings::Deletions::NewView.new(
                current_user: current_user!,
                topic:,
                form:
              ),
              status: :unprocessable_entity
            )
          end

          TopicService::SoftDestroy.new.call(topic_record:)

          flash[:notice] = t("messages.topics.deleted")
          redirect_to space_path(space_record.identifier)
        end

        sig { returns(ActionController::Parameters) }
        private def form_params
          T.cast(params.require(:topic_destroy_confirmation_form), ActionController::Parameters).permit(
            :topic_name
          )
        end
      end
    end
  end
end
