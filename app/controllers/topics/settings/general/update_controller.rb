# typed: true
# frozen_string_literal: true

module Topics
  module Settings
    module General
      class UpdateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable
        include ControllerConcerns::TopicAware

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_policy = space_policy_for(space_record:)
          topic_record = space_policy.showable_topics(space_record:).find_by!(
            number: params[:topic_number]
          )
          topic_policy = topic_policy_for(topic_record:)

          unless topic_policy.can_update_topic?(topic_record:)
            return render_404
          end

          form = Topics::EditForm.new(form_params.merge(topic_record:))

          if form.invalid?
            topic = TopicRepository.new.to_model(topic_record:)

            return render_component(
              Topics::Settings::General::ShowView.new(
                current_user: current_user!,
                topic:,
                form:
              ),
              status: :unprocessable_entity
            )
          end

          Topics::UpdateService.new.call(
            topic_record:,
            name: form.name.not_nil!,
            description: form.description.not_nil!,
            visibility: form.visibility.not_nil!
          )

          flash[:notice] = t("messages.topics.updated")
          redirect_to topic_settings_general_path(space_record.identifier, topic_record.number)
        end

        sig { returns(ActionController::Parameters) }
        private def form_params
          T.cast(params.require(:topics_edit_form), ActionController::Parameters).permit(
            :name,
            :description,
            :visibility
          )
        end
      end
    end
  end
end
