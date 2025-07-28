# typed: strict
# frozen_string_literal: true

class DestroyAccountJob < ApplicationJob
  queue_as :low

  sig { params(user_record_id: T::Wikino::DatabaseId).void }
  def perform(user_record_id:)
    Accounts::DestroyService.new.call(user_record_id:)
  end
end
