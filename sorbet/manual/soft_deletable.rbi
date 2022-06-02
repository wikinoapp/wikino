# typed: strong
# frozen_string_literal: true

module SoftDeletable
  def self.scope(name, body, &block); end
  def self.where(*_arg0, **_arg1, &_arg2); end
  def deleted_at; end
end
