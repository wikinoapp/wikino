# typed: true
# frozen_string_literal: true

class Notes::NewController < ApplicationController
  include Authenticatable

  before_action :authenticate_user

  sig { returns(T.untyped) }
  def call
    result = CreateNoteService.new(form: NoteCreatingForm.new(user: current_user)).call

    if result.errors.any?
      raise StandardError, result.errors.map(&:message).join(", ")
    end

    redirect_to note_path(T.must(result.note).id)
  end
end
