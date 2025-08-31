# typed: true
# frozen_string_literal: true

module Topics
  module Settings
    module Deletions
      class NewController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable
        include ControllerConcerns::TopicAware

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          topic_record = space_record.find_topic_by_number!(params[:topic_number].to_i)
          topic_policy = topic_policy_for(topic_record:)

          unless topic_policy.can_delete_topic?(topic_record:)
            return render_404
          end

          topic = TopicRepository.new.to_model(topic_record:)
          form = Topics::DestroyConfirmationForm.new(topic_record:)

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
