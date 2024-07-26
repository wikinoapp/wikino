# typed: true
# frozen_string_literal: true

module Api
  module Internal
    module Notes
      class UpdateController < ApplicationController
        #   include Authenticatable
        #
        #   before_action :authenticate_user
        #
        #   sig { returns(T.untyped) }
        #   def call
        #     note = T.must(current_user).notes.find(params[:note_id])
        #     form = NoteUpdatingForm.new(user: current_user, note:, body: note_params[:body])
        #     result = UpdateNoteService.new(form:).call
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
        #   def note_params
        #     params.require(:note).permit(:body)
        #   end
      end
    end
  end
end
