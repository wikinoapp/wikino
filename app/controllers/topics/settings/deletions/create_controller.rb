# typed: true
# frozen_string_literal: true

module Topics
  module Settings
    module Deletions
      class CreateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable
        include ControllerConcerns::TopicAware

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          topic_record = space_record.topic_records.kept.find_by!(number: params[:topic_number])
          topic_policy = topic_policy_for(topic_record:)

          unless topic_policy.can_delete_topic?(topic_record:)
            return render_404
          end

          form = Topics::DestroyConfirmationForm.new(form_params.merge(topic_record:))

          if form.invalid?
            topic = TopicRepository.new.to_model(topic_record:)

            return render_component(
              Topics::Settings::Deletions::NewView.new(
                current_user: current_user!,
                topic:,
                form:
              ),
              status: :unprocessable_entity
            )
          end

          Topics::SoftDestroyService.new.call(topic_record:)

          flash[:notice] = t("messages.topics.deleted")
          redirect_to space_path(space_record.identifier)
        end

        sig { returns(ActionController::Parameters) }
        private def form_params
          T.cast(params.require(:topics_destroy_confirmation_form), ActionController::Parameters).permit(
            :topic_name
          )
        end
      end
    end
  end
end
