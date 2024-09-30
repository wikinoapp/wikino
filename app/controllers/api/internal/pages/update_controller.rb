# typed: true
# frozen_string_literal: true

module Api
  module Internal
    module Pages
      class UpdateController < ApplicationController
        #   include Authenticatable
        #
        #   before_action :authenticate_user
        #
        #   sig { returns(T.untyped) }
        #   def call
        #     page = T.must(current_user).pages.find(params[:page_id])
        #     form = PageUpdatingForm.new(user: current_user, page:, body: page_params[:body])
        #     result = UpdatePageService.new(form:).call
        #
        #     if result.errors.any?
        #       return render(
        #         status: :unprocessable_entity,
        #         json: {
        #           messages: result.errors.map(&:message)
        #         }
        #       )
        #     end
        #
        #     head :no_content
        #   end
        #
        #   private
        #
        #   def page_params
        #     params.require(:page).permit(:body)
        #   end
      end
    end
  end
end
