# typed: strict

# DO NOT EDIT MANUALLY
# This file was pulled from https://raw.githubusercontent.com/Shopify/rbi-central/main.
# Please run `bin/tapioca annotations` to update it.

class ActionMailer::Base
  sig { params(headers: T.untyped, block: T.nilable(T.proc.void)).returns(Mail::Message) }
  def mail(headers = nil, &block); end
end