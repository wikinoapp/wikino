# typed: true
# frozen_string_literal: true

module Topics
  module Settings
    module Deletions
      class NewController < ApplicationController
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

          topic = TopicRepository.new.to_model(topic_record:)
          form = TopicForm::DestroyConfirmation.new(topic_record:)

          render_component Topics::Settings::Deletions::NewView.new(
            current_user: current_user!,
            topic:,
            form:
          )
        end
      end
    end
  end
end
