# typed: false
# frozen_string_literal: true

Rails.application.routes.draw do
  if Rails.env.development?
    mount LetterOpenerWeb::Engine, at: "/letter_opener"
  end

  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/accounts",                                               via: :post,   as: :account_list,                  to: "accounts/create#call"
  match "/accounts/new",                                           via: :get,    as: :new_account,                   to: "accounts/new#call"
  match "/email_confirmation",                                     via: :patch,  as: :email_confirmation,            to: "email_confirmations/update#call"
  match "/email_confirmation",                                     via: :post,                                       to: "email_confirmations/create#call"
  match "/email_confirmation/edit",                                via: :get,    as: :edit_email_confirmation,       to: "email_confirmations/edit#call"
  match "/home",                                                   via: :get,    as: :home,                          to: "home/show#call"
  match "/manifest",                                               via: :get,    as: :manifest,                      to: "manifests/show#call"
  match "/s/:space_identifier",                                    via: :get,    as: :space,                         to: "spaces/show#call"
  match "/s/:space_identifier/atom",                               via: :get,    as: :atom,                          to: "atom/show#call"
  match "/s/:space_identifier/bulk_restored_pages",                via: :post,   as: :bulk_restored_page_list,       to: "bulk_restored_pages/create#call"
  match "/s/:space_identifier/settings",                           via: :get,    as: :space_settings,                to: "spaces/settings/show#call"
  match "/s/:space_identifier/pages/:page_number",                 via: :get,    as: :page,                          to: "pages/show#call",                     page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number",                 via: :patch,                                      to: "pages/update#call",                   page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/backlinks",       via: :post,   as: :page_backlink_list,            to: "backlinks/index#call",                page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/draft_page",      via: :patch,  as: :draft_page,                    to: "draft_pages/update#call",             page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/edit",            via: :get,    as: :edit_page,                     to: "pages/edit#call",                     page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/links",           via: :post,   as: :page_link_list,                to: "links/index#call",                    page_number: /\d+/
  match "/s/:space_identifier/pages/:page_number/trash",           via: :post,   as: :trashed_page,                  to: "trashed_pages/create#call",           page_number: /\d+/
  match "/s/:space_identifier/topics",                             via: :post,   as: :topic_list,                    to: "topics/create#call"
  match "/s/:space_identifier/topics/:topic_number",               via: :get,    as: :topic,                         to: "topics/show#call",                    topic_number: /\d+/
  match "/s/:space_identifier/topics/:topic_number/pages/new",     via: :get,    as: :new_page,                      to: "pages/new#call",                      topic_number: /\d+/
  match "/s/:space_identifier/topics/new",                         via: :get,    as: :new_topic,                     to: "topics/new#call"
  match "/s/:space_identifier/trash",                              via: :get,    as: :trash,                         to: "trash/show#call"
  match "/user_session",                                           via: :delete, as: :user_session,                  to: "sessions/destroy#call"
  match "/user_session",                                           via: :post,                                       to: "sessions/create#call"
  match "/sign_in",                                                via: :get,    as: :sign_in,                       to: "sign_in/show#call"
  match "/sign_up",                                                via: :get,    as: :sign_up,                       to: "sign_up/show#call"
  match "/spaces",                                                 via: :post,   as: :space_list,                    to: "spaces/create#call"
  match "/spaces/new",                                             via: :get,    as: :new_space,                     to: "spaces/new#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute

  root "welcome/show#call"
end
