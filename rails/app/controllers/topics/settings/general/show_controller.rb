# typed: true
# frozen_string_literal: true

module Topics
  module Settings
    module General
      class ShowController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable
        include ControllerConcerns::TopicAware

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          topic_record = space_record.topic_record_by_number!(params[:topic_number])
          topic_policy = topic_policy_for(topic_record:)

          unless topic_policy.can_update_topic?(topic_record:)
            return render_404
          end

          form = Topics::EditForm.new(
            name: topic_record.name,
            description: topic_record.description,
            visibility: topic_record.visibility
          )
          topic = TopicRepository.new.to_model(topic_record:)

          render_component Topics::Settings::General::ShowView.new(
            current_user: current_user!,
            topic:,
            form:
          )
        end
      end
    end
  end
end
